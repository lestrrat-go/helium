<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error
                     if two or more sibling xsl:with-param elements have
                        name attributes that represent the same expanded QName.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE0670?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates>
            <xsl:with-param name="my:p" select="2"/>
            <xsl:with-param name="q" select="2"/>
            <xsl:with-param name="r" select="2"/>
            <xsl:with-param xmlns:your="http://my.com/" name="your:p" select="2"/>
        </xsl:apply-templates>  
      </out>
  </xsl:template>
  

  


</xsl:stylesheet>
