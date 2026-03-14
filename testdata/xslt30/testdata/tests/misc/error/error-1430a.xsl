<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error if there is no
                        namespace bound to the prefix on the element bearing the
                           [xsl:]extension-element-prefixes attribute or, when
                           #default is specified, if there is no default namespace.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1430?>

  <xsl:param name="p" select="'123'"/>


  <xsl:template name="main">
    
      <out>        
          <e xsl:extension-element-prefixes="my your"/>     
      </out>
  </xsl:template>
  



</xsl:stylesheet>
