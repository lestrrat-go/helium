<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:myfn="urn:local" version="2.0">
    <xsl:template name="DropIfEmpty">
        <xsl:variable name="res">
            <!-- Oxygen reports exception here.  -->
            <xsl:next-match/>
        </xsl:variable>
        <xsl:if test="$res/*[* or (normalize-space(string-join(text(),'')) ne '')]">
            <xsl:sequence select="$res"/>
        </xsl:if>
    </xsl:template>
    
    <!-- high priority rule to match element(elem) -->
    <xsl:template match="elem" priority="10">
        <xsl:call-template name="DropIfEmpty"/>
    </xsl:template>
    
    <!-- A low priority rule that matches any element whose name starts with 'other'.  This won't actually match anything in the input document. -->
    <xsl:template match="*[myfn:is-other(.)]" priority="5"/>
    
    <!-- A function that can be used in the above template. --> 
    <xsl:function name="myfn:is-other">
        <xsl:param name="elem" as="element()"/>
        <xsl:sequence select="starts-with($elem/local-name(), 'other')"/>
    </xsl:function>
    
    <xsl:template match="/">
        <out>
            <xsl:apply-templates/>
        </out>
    </xsl:template>
</xsl:stylesheet>
