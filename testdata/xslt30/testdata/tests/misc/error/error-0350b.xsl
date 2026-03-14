<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an unescaped
                     left curly bracket appears in a fixed part of a content value template
                     without a matching right curly bracket.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:my="http://my.uri/"
                version="2.0">

<?error XTSE0350?>



  <xsl:template name="main">
      <out>
         <e att="{banana"/>
      </out>
  </xsl:template>
  


</xsl:stylesheet>
