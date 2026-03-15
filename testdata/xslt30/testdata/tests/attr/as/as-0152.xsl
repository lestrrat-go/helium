<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                xmlns:f="f"
                exclude-result-prefixes="xs"
                version="2.0">

   <!-- Purpose: Test promotion of function result to xs:double  -->

   <xsl:function name="f:promote" as="xs:double">
      <xsl:param name="p"/>
      <xsl:sequence select="$p"/>
   </xsl:function>
   
   <xsl:param name="base" select="0" as="xs:integer"/>
   <xsl:param name="test" select="0.0e0"/>
   
   <xsl:template name="xsl:initial-template">
      <out>
         <a result="{(f:promote($base + 12), $test) instance of xs:double*}"/>
         <b result="{(f:promote($base + 12.2), $test) instance of xs:double*}"/>
         <c result="{(f:promote($base + xs:float(0.5)), $test) instance of xs:double*}"/>
         <d result="{(f:promote(xs:untypedAtomic($base || '1.123')), $test) instance of xs:double*}"/>
      </out>
   </xsl:template>
</xsl:transform>
