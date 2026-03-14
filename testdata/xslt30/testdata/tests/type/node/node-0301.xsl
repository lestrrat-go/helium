<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#node-tests?>
    <!-- Purpose: Test for processing-instruction() node test. -->

<!-- should say "Found-pi,,Found-pi" -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="./processing-instruction('a-pi')"/>
    <!--<xsl:apply-templates select="./processing-instruction('*')"/>-->   
    <xsl:apply-templates select="./processing-instruction()"/>	
  </out>
</xsl:template>

<xsl:template match="processing-instruction('a-pi')">
  <xsl:text>Found-pi:</xsl:text>
  <xsl:value-of select="name()"/>
  <xsl:copy/>
</xsl:template>

</xsl:stylesheet>
