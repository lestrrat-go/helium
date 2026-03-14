<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- FileName: copy2112 -->
    <!-- Purpose: Use copy-of() to copy an element node. No arguments. -->


<xsl:template match="/">
  <out>
    <xsl:sequence select="/doc/*/copy-of()"/>
  </out>
</xsl:template>

</xsl:stylesheet>