<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <!-- Purpose: Test for processing-instruction() node-test in copy-of. -->



<xsl:template match="/">
  <out>
    <xsl:copy-of select="./processing-instruction('a-pi')"/>;
    <xsl:copy-of select="./processing-instruction(pi-2)"/>;   
    <xsl:copy-of select="./processing-instruction()"/>
  </out>
</xsl:template>

</xsl:stylesheet>
