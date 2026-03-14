<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<xsl:import href="include-0701b.xsl"/>
<xsl:import href="include-0701c.xsl"/>

<xsl:template match="/">
    <out>
      <xsl:apply-templates/>
    </out>
</xsl:template>
 
<xsl:template match="title">
  Import01-title<xsl:text>,</xsl:text>
  <xsl:apply-imports/>
</xsl:template>

</xsl:stylesheet>
