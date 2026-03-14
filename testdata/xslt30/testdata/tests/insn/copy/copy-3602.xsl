<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: copy53 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 11.3 -->
    <!-- Purpose: Copy all PIs of a certain name via copy-of. -->

<xsl:template match="/">
  <out>
    <xsl:copy-of select="//processing-instruction('b-pi')"/>
  </out>
</xsl:template>

</xsl:stylesheet>