<?xml version="1.0" encoding="UTF-8"?>
<!--It is a non-recoverable dynamic
                        error if the effective
                        value of the name attribute of the
                           xsl:element instruction is not a lexical QName. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:element - bad prefix in QName -->
<?spec xslt#errors?><?error XTDE0820?>


  <xsl:template match="doc" name="this">
      <out>text
      <xsl:element name="1prefix:name" namespace="http://a.com/uri">
            <xsl:attribute name="nasty">value</xsl:attribute>
         </xsl:element>  
         <xsl:message>Error not detected (or recovered)!</xsl:message>
      </out>
  </xsl:template>
  

</xsl:stylesheet>
