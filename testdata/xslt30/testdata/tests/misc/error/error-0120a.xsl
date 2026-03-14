<?xml version="1.0" encoding="UTF-8"?>
<!--An xsl:stylesheet element must not have any
                     text node children.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: text at top level -->
<?spec xslt#errors?><?error XTSE0010?>

  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

text
<xsl:variable name="z" select="16"/>

</xsl:stylesheet>
