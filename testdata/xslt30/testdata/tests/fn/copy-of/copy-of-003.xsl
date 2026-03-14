<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- FileName: copy2114 -->
    <!-- Purpose: Use copy-of() to copy a comment node; and check different node identity -->


<xsl:template match="/">
  <out>
    <xsl:sequence select=".//comment() is copy-of(.//comment())"/>
  </out>
</xsl:template>

</xsl:stylesheet>