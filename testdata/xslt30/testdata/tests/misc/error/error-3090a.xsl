<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0"
  xmlns:f="http://example.com/f">
              <!--It is a type error if the xsl:context-item
                  child of xsl:template specifies that a context item is required and none is supplied
                  by the caller, that is, if the context item is absent at the point where xsl:call-template
                  is evaluated.-->
   <xsl:template name="main">
      <out>
         <xsl:value-of select="f:f()"/>
      </out>
   </xsl:template>
   
   <xsl:function name="f:f">
     <xsl:call-template name="t"/>
   </xsl:function>
   
   <xsl:template name="t">
     <xsl:context-item use="required"/>
     <xsl:value-of select="."/>
   </xsl:template>
</xsl:stylesheet>
