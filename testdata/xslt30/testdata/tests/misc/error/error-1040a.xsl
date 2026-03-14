<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if an
                        xsl:perform-sort instruction with a select
                     attribute has any content other than xsl:sort and
                        xsl:fallback instructions.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1040?>


  <xsl:template name="main">
      <out>
         <xsl:perform-sort select="1 to 5">
            <xsl:sort select="."/>
            <xsl:value-of select="."/>
         </xsl:perform-sort>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
