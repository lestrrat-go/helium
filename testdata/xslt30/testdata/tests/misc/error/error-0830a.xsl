<?xml version="1.0" encoding="UTF-8"?>
<!--In the case of an xsl:element instruction with no
                        namespace attribute, it is a non-recoverable dynamic error if the
                        effective value of the
                        name attribute is a lexical QName
                     whose prefix is not declared in an in-scope namespace declaration for the
                        xsl:element instruction.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:element  -->
<?spec xslt#errors?><?error XTDE0830?>


  <xsl:template match="doc" name="this">
      <out>
         <xsl:element name="prefix:name"/>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  

</xsl:stylesheet>
