<?xml version="1.0" encoding="UTF-8"?>
<!--When the group-starting-with or group-ending-with
                     attribute of the xsl:for-each-group
                        instruction is used, it is a type
                        error if the result of evaluating the select
                     expression contains an item that is not a node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://my.com/"
                version="2.0">

<?error XTTE1120?>


  <xsl:template name="main">
      <xsl:variable name="pop">
         <a/>
         <b/>
         <c/>
      </xsl:variable>
      <out>
         <xsl:for-each-group select="1 to 5" group-starting-with="node()">
            <xsl:value-of select="current-group()"/>
         </xsl:for-each-group>
      </out>
  </xsl:template>
  



</xsl:stylesheet>
