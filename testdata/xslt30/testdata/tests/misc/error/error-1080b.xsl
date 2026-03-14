<?xml version="1.0" encoding="UTF-8"?>
<!--These four attributes the group-by,
                           group-adjacent, group-starting-with, and
                           group-ending-with attributes of
                           xsl:for-each-group
                      are mutually exclusive: it is a static error if none of these four attributes is present or if
                     more than one of them is present.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTSE1080?>


  <xsl:template name="main">
      <out>
         <xsl:for-each-group select="1 to 5" group-by="." group-adjacent=".">
            <xsl:value-of select="current-group()"/>
         </xsl:for-each-group>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
