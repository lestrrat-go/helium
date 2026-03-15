<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if the list
                        of modes in the mode attribute of
                              xsl:template
                         is empty, if the same token is included more than once in the
                        list, if the list contains an invalid token, or if the token
                           #all appears together with any other value.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0">

<?error XTSE0550?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates/>
      </out>
  </xsl:template>
  
  <xsl:template match="doc" mode="a b c# a">
      <a/>
  </xsl:template>
  

  


</xsl:stylesheet>
