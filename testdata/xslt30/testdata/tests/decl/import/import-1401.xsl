<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: When no stylesheets are imported, an xsl:apply-imports should
       select the built-in templates. -->

<xsl:template match="/">
  <result>
    Before apply-imports
      <xsl:apply-imports/>
    After apply-imports
  </result>
</xsl:template>

</xsl:stylesheet>
