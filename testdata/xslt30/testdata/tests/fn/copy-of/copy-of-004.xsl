<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

  <!-- FileName: copy2115 -->
    <!-- Purpose: Use copy-of() to copy a processing instruction; and check it has no siblings -->


<xsl:template match="/">
  <out>
    <xsl:sequence select="count(copy-of(.//processing-instruction())/preceding-sibling::node())"/>
  </out>
</xsl:template>

</xsl:stylesheet>