<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the
               current-group function is used within a pattern.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1060?>


  <xsl:template name="main">
      <out>
         <xsl:number count="*[empty(current-group())]"/>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
