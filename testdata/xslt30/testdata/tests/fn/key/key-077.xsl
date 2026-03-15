<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
  
  <!-- Test that xml:id is recognized in a temporary tree -->
  
  <xsl:variable name="lookup">
    <code xml:id="a">Apple</code>
    <code xml:id="b">Banana</code>
    <code xml:id="c">Cherry</code>
    <code xml:id="d">Damson</code>
    <code xml:id="e">Elderberry</code>
    <code xml:id="f">Fraise</code>
  </xsl:variable>          

  <xsl:template match="main">
    <out><xsl:value-of select="id('d', $lookup)"/></out>
  </xsl:template>
  

</xsl:stylesheet>
