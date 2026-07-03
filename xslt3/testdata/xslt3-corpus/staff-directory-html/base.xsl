<?xml version="1.0" encoding="UTF-8"?>
<!-- Imported base module: a low-import-precedence person template that renders
     just the person's name. The main module overrides it and reuses this
     rendering via xsl:apply-imports. -->
<xsl:stylesheet version="3.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

  <xsl:template match="person">
    <span class="name"><xsl:value-of select="@name"/></span>
  </xsl:template>

</xsl:stylesheet>
