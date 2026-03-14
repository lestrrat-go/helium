<?xml version="1.0" encoding="UTF-8"?>
<!--Where an attribute is defined to contain a pattern, it is a static
                           error if the pattern does not match the production Pattern.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:my="http://my.uri/"
                version="2.0">

<?error XTSE0340?>



  <xsl:template name="main">
      <out>
         <xsl:number count="2+2"/>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
