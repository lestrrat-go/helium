<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a stylesheet contains more than one binding of a
                     global variable with the same name and same import precedence, unless it also contains another binding with
                     the same name and higher import precedence.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:variable - duplicate name -->
<?spec xslt#errors?><?error XTSE0630?>

  <xsl:variable name="x" select="1"/>
  <xsl:variable name="x" select="2"/>
 
  <xsl:template match="doc">

      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>

</xsl:stylesheet>
