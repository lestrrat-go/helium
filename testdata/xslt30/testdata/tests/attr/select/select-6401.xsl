<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Try /{name} to match the document element. -->

<xsl:template match="/Q">
  <out>
    <xsl:value-of select="@level"/>
  </out>
</xsl:template>

</xsl:stylesheet>