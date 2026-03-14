<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    xmlns:math="http://www.w3.org/2005/xpath-functions/math" 
    exclude-result-prefixes="xs math" 
    version="3.0">
    
    <!-- Merge keys are evaluated with a singleton focus, therefore we end up with a single group having grouping key = 1 -->
    
    <xsl:param name="doc1">
        <root>
            <foo>foo 1</foo>
            <bar>bar 1</bar>
            <bar>bar 2</bar>
            <foo>foo 2</foo>
            <baz>baz 1</baz>
        </root>
    </xsl:param>
    
    <xsl:param name="doc2">
        <root>
            <foo>foo a</foo>
            <bar>bar a</bar>
            <bar>bar b</bar>
            <foo>foo b</foo>
            <baz>baz a</baz>
        </root>
    </xsl:param>

    
    <xsl:template name="xsl:initial-template" match="/">
        <root>
            <xsl:merge>
                <xsl:merge-source select="$doc1/*/*">
                    <xsl:merge-key select="position()"></xsl:merge-key>
                </xsl:merge-source>
                <xsl:merge-source select="$doc2/*/*">
                    <xsl:merge-key select="position()"></xsl:merge-key>
                </xsl:merge-source>
                <xsl:merge-action>
                    <group key="{current-merge-key()}">
                        <xsl:copy-of select="current-merge-group()"/>
                    </group>
                </xsl:merge-action>
            </xsl:merge>
        </root>
    </xsl:template>
    
</xsl:stylesheet>