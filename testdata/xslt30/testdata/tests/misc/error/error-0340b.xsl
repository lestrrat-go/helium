<?xml version="1.0" encoding="UTF-8"?>
<!--Where an attribute is defined to contain a pattern, it is a static
                           error if the pattern does not match the production Pattern.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:template - bad pattern -->
<?spec xslt#errors?><?error XTSE0340?>

   <xsl:template match="name/1223"/>
  
  <xsl:template match="doc">
      <out>
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
