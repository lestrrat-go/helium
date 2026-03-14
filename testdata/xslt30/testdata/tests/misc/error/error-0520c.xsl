<?xml version="1.0" encoding="UTF-8"?>
<!--It is a type error if the sequence
                     returned by the select expression of
                           xsl:apply-templates
                      contains an item that is not a node.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:xs="http://www.w3.org/2001/XMLSchema"
                version="2.0"
                exclude-result-prefixes="xs">

<!-- PURPOSE:  "." used when not a node - optionally a static error even though never executed -->
<?spec xpath#errors?><?error XTTE0520?>

   <xsl:template match="/">
      <a>***** error not detected! *****</a>
   </xsl:template>

   <xsl:function xmlns:f="http://f.com/" name="f:doc">
      <xsl:for-each select="1 to 5">
         <xsl:apply-templates select="."/>
      </xsl:for-each>
   </xsl:function>
</xsl:stylesheet>
