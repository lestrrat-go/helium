<xsl:stylesheet version="1.0"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
   xmlns:xs="http://www.w3.org/2001/XMLSchema">

   <xsl:template name="xsl:initial-template">
      <out>
         <xsl:call-template name="test">
            <xsl:with-param name="arg" select="()"/>
         </xsl:call-template>
      </out>
   </xsl:template>
   
   <xsl:template name="test">
      <xsl:param name="arg" as="xs:string?"/>
      <result count="{count($arg)}" value="{$arg}"/>
   </xsl:template>


</xsl:stylesheet>
