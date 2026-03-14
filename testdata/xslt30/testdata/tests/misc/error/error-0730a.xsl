<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

              <!--If an xsl:attribute set element specifies streamable="yes" then every attribute set referenced in its
                  use-attribute-sets attribute (if present) must also specify streamable="yes".-->
                  
   <xsl:attribute-set name="a" streamable="yes" use-attribute-sets="b c">
     <xsl:attribute name="x" select="1"/>
   </xsl:attribute-set>
   
   <xsl:attribute-set name="b" streamable="yes">
     <xsl:attribute name="y" select="1"/>
   </xsl:attribute-set>
   
   <xsl:attribute-set name="c" streamable="0">
     <xsl:attribute name="z" select="1"/>
   </xsl:attribute-set>
   
   <xsl:template name="main">
      <out>
         <x/>
      </out>
   </xsl:template>
</xsl:stylesheet>
