<?xml version="1.0" encoding="UTF-8"?>

<!-- test xsl:key with decimal keys -->


<xsl:stylesheet version="2.0" 
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" 
    exclude-result-prefixes="xs">

<xsl:key name="i" match="item" use="xs:decimal(.)"/>

<xsl:template match="/">
    <out><xsl:value-of select="key('i', 10)/@id" separator=","/></out>
</xsl:template>
</xsl:stylesheet>
