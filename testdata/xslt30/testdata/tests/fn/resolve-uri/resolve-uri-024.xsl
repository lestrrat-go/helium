<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    version="3.0" expand-text="yes">

    <!-- 
        Extracts static-base-uri tests from resolve-uri-022 for ease of analysis
    -->
    

    <xsl:output indent="no"></xsl:output>

    <xsl:template name="initial-template">
        
        <out xml:base="http://example.com/a/b/c/d/../d/e/f/g/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z">
            <xsl:variable name="base" select="static-base-uri()" xml:base="Z/../Z"/>
            
            <!-- varying bases -->
            <base1>{$base}</base1>
            <base2>{static-base-uri()}</base2>
            <base3>{resolve-uri(static-base-uri(), static-base-uri())}</base3>
        </out>
    </xsl:template>

</xsl:stylesheet>
