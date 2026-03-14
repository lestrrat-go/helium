<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
    xmlns:xs="http://www.w3.org/2001/XMLSchema"
    xmlns:f="http://local/"
    exclude-result-prefixes="xs f"
    version="3.0">
    
    <!-- apply-templates to a function item -->
    
    <xsl:template match=".[ . instance of function(*)]">
      <f name="{local-name-from-QName(function-name(.))}" arity="{function-arity(.)}"/>
    </xsl:template>
    
    <xsl:function name="f:squared">
      <xsl:param name="x"/>
      <xsl:sequence select="$x * $x"/>
    </xsl:function>
    
    <xsl:template name="xsl:initial-template">
        <out>
          <xsl:apply-templates select="round#1"/>
          <xsl:apply-templates select="f:squared#1"/>
          <xsl:apply-templates select="contains(?, 'e')"/>
          <xsl:apply-templates select="function($x){$x+$x}"/>
        </out>
    </xsl:template>
    
</xsl:stylesheet>