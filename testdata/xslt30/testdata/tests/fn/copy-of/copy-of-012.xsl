<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
    xmlns:f="http://f.org/"
    exclude-result-prefixes="f">

  <!-- FileName: copy2198 -->
    <!-- Purpose: Use copy-of() with no arguments when there is no context node -->


<xsl:template match="/">
  <out>
    <xsl:sequence select="f:copy(.)"/>
  </out>
</xsl:template>

<xsl:function name="f:copy">
    <xsl:param name="n"/>
    <xsl:sequence select="copy-of()"/>
</xsl:function>

</xsl:stylesheet>