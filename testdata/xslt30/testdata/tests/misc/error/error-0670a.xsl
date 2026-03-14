<?xml version="1.0" encoding="UTF-8"?>
<!--It is a static error
                     if two or more sibling xsl:with-param elements have
                        name attributes that represent the same expanded QName.-->
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- Error: xsl:call-template - duplicate parameters -->
<?spec xslt#errors?><?error XTSE0670?>

   <xsl:template name="x">
      <xsl:param name="y"/>
   </xsl:template>
  
  <xsl:template match="doc">
      <out>
         <xsl:call-template name="x">
            <xsl:with-param name="y" select="3"/>
            <xsl:with-param name="y" select="4"/>
         </xsl:call-template>  
         <xsl:message>Error not detected!</xsl:message>
      </out>
  </xsl:template>




</xsl:stylesheet>
