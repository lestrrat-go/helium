<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:map="http://www.w3.org/2005/xpath-functions/map"
    exclude-result-prefixes="#all"
    version="3.0">
    
    <!-- Saxon bug 3257 -->
    
    <!-- Builds a map containing word-pairs from the source document, and generates an output document by
        generating words from this set of word-pairs chosen at random. -->
    
    <xsl:mode streamable="yes"/>
    
    <xsl:template match="/">
        <xsl:variable name="words" 
                      as="xs:string*" 
                      select="//text()!tokenize(., '\s+')[matches(., '^[a-zA-Z]+$')]"/>
        <xsl:variable name="histogram" as="map(*)">
            <xsl:iterate select="tail($words)">
                <xsl:param name="previous" select="head($words)"/>
                <xsl:param name="histogram" select="map{}"/>
                <xsl:on-completion select="$histogram"/>
                <xsl:variable name="new-entry" select="if (map:contains($histogram, $previous)) then (map:get($histogram, $previous), .) else (.)"/>
                <xsl:choose>
                    <xsl:when test="map:size($histogram) gt 1000">
                        <!-- artificial limit to make the test faster -->
                        <xsl:break select="$histogram"/>
                    </xsl:when>
                    <xsl:otherwise>
                        <xsl:next-iteration>
                            <xsl:with-param name="previous" select="."/>
                            <xsl:with-param name="histogram" select="map:put($histogram, $previous, $new-entry)"/>
                        </xsl:next-iteration>
                    </xsl:otherwise>
                </xsl:choose>               
            </xsl:iterate>
        </xsl:variable>
        <p>
            <xsl:text>And</xsl:text>
            <xsl:iterate select="1 to 200">
                <xsl:param name="word" select="'And'"/>
                <xsl:variable name="options" select="map:get($histogram, $word), 'the'"/>
                <xsl:variable name="next" select="(random-number-generator(.)?permute($options))[1]"/>
                <xsl:value-of select="(if (matches($next, '^[A-Z]')) then '. ' else ' ') || $next"/>
                <xsl:next-iteration>
                    <xsl:with-param name="word" select="$next"/>
                </xsl:next-iteration>
            </xsl:iterate>
            <xsl:text> covfefe.</xsl:text>
        </p>
    </xsl:template>
 
</xsl:stylesheet>