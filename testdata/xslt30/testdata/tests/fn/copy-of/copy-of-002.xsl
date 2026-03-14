<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- FileName: copy2113 -->
    <!-- Purpose: Use copy-of() to copy an element node. One argument. -->


<xsl:template match="/">
  <out>
    <xsl:sequence select="for $e in /doc/* return copy-of($e)"/>
  </out>
</xsl:template>

</xsl:stylesheet>