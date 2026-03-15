<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the result of
                     evaluating the group-adjacent expression is an empty sequence
                     or a sequence containing more than one item, 
                     unless composite="yes" is specified.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1100?>


  <xsl:template name="main">
      <xsl:variable name="pop">
         <a/>
         <b/>
         <c/>
      </xsl:variable>
      <out>
         <xsl:for-each-group select="$pop/*" group-adjacent="1,2,3">
            <xsl:value-of select="current-group()"/>
         </xsl:for-each-group>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
