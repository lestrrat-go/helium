<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an unescaped
                     right curly bracket occurs in a fixed part of a content value template.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:my="http://my.uri/"
                version="2.0">

<?error XTSE0370?>



  <xsl:template name="main">
      <out>
         <e att="{2+2} and }3"/>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
