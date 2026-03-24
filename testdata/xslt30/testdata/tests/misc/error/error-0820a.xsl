<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the name attribute of the
                           xsl:element instruction is not a lexical QName. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:element bad element name (this was a recoverable error) -->
<?spec xslt#errors?><?error XTDE0820?>


  <xsl:template match="doc" name="this">
      <out>
         <xsl:element name="a bad name"/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  

</xsl:stylesheet>
