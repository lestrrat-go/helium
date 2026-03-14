<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform" 
    xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="local-functions.uri" 
    exclude-result-prefixes="xs f">

<xsl:output method="xml" version="1.0" encoding="UTF-8" indent="yes"/>

<xsl:function name="f:sum" expand-text="yes">
        <xsl:param name="x" as="xs:integer"/>
        <xsl:param name="y" as="xs:integer"/>
  {$x + $y}
</xsl:function>


<xsl:template match="/" name="main">
        <out>
            <xsl:value-of select="f:sum(1,2) instance of text()"/>
        </out>
</xsl:template>

</xsl:stylesheet>