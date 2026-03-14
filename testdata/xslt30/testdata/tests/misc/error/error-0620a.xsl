<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a variable-binding element has a
                        select attribute and has non-empty content.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:variable - select attribute plus content  -->
<?spec xslt#errors?><?error XTSE0620?>

  <xsl:template match="doc">
      <out>
         <xsl:variable name="x" select="1">text</xsl:variable>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>
