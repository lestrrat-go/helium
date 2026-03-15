<?xml version="1.0"?>
<xsl:stylesheet version="3.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    exclude-result-prefixes="xs">

<!-- Type error when xsl:evaluate returns wrong type  -->

    <xsl:param name="path" select="'2+2'"/>

    <xsl:template match="/">
        <out>
            <xsl:evaluate xpath="$path" as="xs:string"/>
        </out>
    </xsl:template>
</xsl:stylesheet>

