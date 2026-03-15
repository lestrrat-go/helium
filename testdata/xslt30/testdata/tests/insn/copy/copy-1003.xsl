<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy43 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.3 -->
    <!-- Purpose: Copy the whole input tree to non-root position in output.
     "a root node is copied by copying its children" -->

<xsl:template match="/">
  <out>
    <xsl:copy-of select="/" />
  </out>
</xsl:template>

</xsl:stylesheet>