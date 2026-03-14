<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- PURPOSE:   Test xsl:sequence adding atomic values to an attribute -->

<xsl:template match="doc">
    <zzz>
      <xsl:attribute name="a">
        <xsl:text>[</xsl:text>
        <xsl:sequence select="1 to 5"/>
        <xsl:text>]</xsl:text>
      </xsl:attribute>
    </zzz>
</xsl:template>
 
</xsl:stylesheet>
