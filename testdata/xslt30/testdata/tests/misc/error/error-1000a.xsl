<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the result of
                     evaluating the select attribute of the xsl:number
                     instruction is anything other than a single node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1000?>


  <xsl:template name="main">
      <out>
         <xsl:for-each select="1 to 5">
            <xsl:number select="17"/>
         </xsl:for-each>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
