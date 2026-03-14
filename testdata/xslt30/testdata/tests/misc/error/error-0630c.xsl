<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if a stylesheet contains more than one binding of a
                     global variable with the same name and same import precedence, unless it also contains another binding with
                     the same name and higher import precedence.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0630?>

  <xsl:param name="glob" select="3"/>   

  <xsl:template name="main">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
  <xsl:variable name="glob" select="2"/>   

  


</xsl:stylesheet>
