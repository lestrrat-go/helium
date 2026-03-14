<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the sequence
                     returned by the select expression of
                           xsl:apply-templates
                      contains an item that is not a node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0">

<?error XTTE0520?>



  <xsl:template name="main">
      <out>
        <xsl:apply-templates select="concat('a', 'b')"/>
      </out>
  </xsl:template>
  
 
  


</xsl:stylesheet>
