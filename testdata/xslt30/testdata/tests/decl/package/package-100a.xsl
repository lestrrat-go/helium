<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:csv="http://example.com/csv"
    exclude-result-prefixes="xs csv" version="3.0">

    <xsl:output indent="yes"/>

    <!-- example input "file"  -->
    <xsl:variable name="input" as="xs:string"
        >name,id,postal code 
        "Abel Braaksma",34291,1210 KA
        "Anders Berglund",473892,9843 ZD</xsl:variable>

    <!-- entry point -->
    <xsl:template name="xsl:initial-template">
        <xsl:copy-of select="csv:parse($input)"/>
    </xsl:template>

    <xsl:use-package name="http://example.com/csv-parser" package-version="*"/>


</xsl:stylesheet>
