<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if an
                        xsl:apply-templates instruction with no select
                     attribute is evaluated when the context
                        item is not a node. -->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0">

<?error XTTE0510?>



  <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:apply-templates/>
         </xsl:for-each>
      </out>
  </xsl:template>
  
 
  


</xsl:stylesheet>
