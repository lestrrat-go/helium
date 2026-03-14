<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs" version="2.0">
   <!-- Purpose: Test with global xsl:variable where @select is a typed value obtained at run time 
  				and @as="xs:anyAtomicType". Effective global variable type is both xs:anyAtomicType 
  				and the actual type used in @select. No Schema is imported. Types tested:
				-all primitive built-in types, xs:integer, all derived builtin types -->

   <xslt:variable name="var1" select="/doc-schemaas/elem-anyURI" as="xs:anyAtomicType"/>

   <xslt:variable name="var2" select="/doc-schemaas/elem-base64Binary" as="xs:anyAtomicType"/>

   <xslt:variable name="var3" select="/doc-schemaas/elem-boolean" as="xs:anyAtomicType"/>

   <xslt:variable name="var4" select="/doc-schemaas/elem-date" as="xs:anyAtomicType"/>

   <xslt:variable name="var5" select="/doc-schemaas/elem-dateTime" as="xs:anyAtomicType"/>

   <xslt:variable name="var6" select="/doc-schemaas/elem-decimal" as="xs:anyAtomicType"/>

   <xslt:variable name="var7" select="/doc-schemaas/elem-double" as="xs:anyAtomicType"/>

   <xslt:variable name="var8" select="/doc-schemaas/elem-duration" as="xs:anyAtomicType"/>

   <xslt:variable name="var9" select="/doc-schemaas/elem-float" as="xs:anyAtomicType"/>

   <xslt:variable name="var10" select="/doc-schemaas/elem-gDay" as="xs:anyAtomicType"/>

   <xslt:variable name="var11" select="/doc-schemaas/elem-gMonth" as="xs:anyAtomicType"/>

   <xslt:variable name="var12" select="/doc-schemaas/elem-gMonthDay" as="xs:anyAtomicType"/>

   <xslt:variable name="var13" select="/doc-schemaas/elem-gYear" as="xs:anyAtomicType"/>

   <xslt:variable name="var14" select="/doc-schemaas/elem-gYearMonth" as="xs:anyAtomicType"/>

   <xslt:variable name="var15" select="/doc-schemaas/elem-hexBinary" as="xs:anyAtomicType"/>

   <xslt:variable name="var16" select="/doc-schemaas/elem-integer" as="xs:anyAtomicType"/>

   <xslt:variable name="var17" select="/doc-schemaas/elem-QName" as="xs:anyAtomicType"/>

   <xslt:variable name="var18" select="/doc-schemaas/elem-string" as="xs:anyAtomicType"/>

   <xslt:variable name="var19" select="/doc-schemaas/elem-time" as="xs:anyAtomicType"/>

   <xslt:variable name="var20" select="/doc-schemaas/elem-normalizedString" as="xs:anyAtomicType"/>

   <xslt:variable name="var21" select="/doc-schemaas/elem-token" as="xs:anyAtomicType"/>

   <xslt:variable name="var22" select="/doc-schemaas/elem-language" as="xs:anyAtomicType"/>

   <xslt:variable name="var23" select="/doc-schemaas/elem-Name" as="xs:anyAtomicType"/>

   <xslt:variable name="var24" select="/doc-schemaas/elem-NCName" as="xs:anyAtomicType"/>

   <xslt:variable name="var25" select="/doc-schemaas/elem-NMTOKEN" as="xs:anyAtomicType"/>

   <xslt:variable name="var26" select="/doc-schemaas/elem-int" as="xs:anyAtomicType"/>

   <xslt:variable name="var27" select="/doc-schemaas/elem-nonPositiveInteger" as="xs:anyAtomicType"/>

   <xslt:variable name="var28" select="/doc-schemaas/elem-positiveInteger" as="xs:anyAtomicType"/>

   <xslt:variable name="var29" select="/doc-schemaas/elem-nonNegativeInteger" as="xs:anyAtomicType"/>

   <xslt:variable name="var30" select="/doc-schemaas/elem-negativeInteger" as="xs:anyAtomicType"/>

   <xslt:variable name="var31" select="/doc-schemaas/elem-long" as="xs:anyAtomicType"/>

   <xslt:variable name="var32" select="/doc-schemaas/elem-unsignedLong" as="xs:anyAtomicType"/>

   <xslt:variable name="var33" select="/doc-schemaas/elem-unsignedInt" as="xs:anyAtomicType"/>

   <xslt:variable name="var34" select="/doc-schemaas/elem-short" as="xs:anyAtomicType"/>

   <xslt:variable name="var35" select="/doc-schemaas/elem-unsignedShort" as="xs:anyAtomicType"/>

   <xslt:variable name="var36" select="/doc-schemaas/elem-byte" as="xs:anyAtomicType"/>

   <xslt:variable name="var37" select="/doc-schemaas/elem-unsignedByte" as="xs:anyAtomicType"/>

   <xslt:variable name="var38" select="/doc-schemaas/elem-ID-attr/@attr1" as="xs:anyAtomicType"/>

   <xslt:variable name="var39" select="/doc-schemaas/elem-IDREF-attr/@attr1" as="xs:anyAtomicType"/>

   <xslt:template match="/doc-schemaas">
      <out>
         <var1>
            <xslt:value-of select="$var1 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var1 instance of xs:anyURI"/>
         </var1>
         <var2>
            <xslt:value-of select="$var2 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var2 instance of xs:base64Binary"/>
         </var2>
         <var3>
            <xslt:value-of select="$var3 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var3 instance of xs:boolean"/>
         </var3>
         <var4>
            <xslt:value-of select="$var4 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var4 instance of xs:date"/>
         </var4>
         <var5>
            <xslt:value-of select="$var5 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var5 instance of xs:dateTime"/>
         </var5>
         <var6>
            <xslt:value-of select="$var6 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var6 instance of xs:decimal"/>
         </var6>
         <var7>
            <xslt:value-of select="$var7 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var7 instance of xs:double"/>
         </var7>
         <var8>
            <xslt:value-of select="$var8 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var8 instance of xs:duration"/>
         </var8>
         <var9>
            <xslt:value-of select="$var9 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var9 instance of xs:float"/>
         </var9>
         <var10>
            <xslt:value-of select="$var10 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var10 instance of xs:gDay"/>
         </var10>
         <var11>
            <xslt:value-of select="$var11 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var11 instance of xs:gMonth"/>
         </var11>
         <var12>
            <xslt:value-of select="$var12 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var12 instance of xs:gMonthDay"/>
         </var12>
         <var13>
            <xslt:value-of select="$var13 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var13 instance of xs:gYear"/>
         </var13>
         <var14>
            <xslt:value-of select="$var14 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var14 instance of xs:gYearMonth"/>
         </var14>
         <var15>
            <xslt:value-of select="$var15 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var15 instance of xs:hexBinary"/>
         </var15>
         <var16>
            <xslt:value-of select="$var16 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var16 instance of xs:integer"/>
         </var16>
         <var17>
            <xslt:value-of select="$var17 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var17 instance of xs:QName"/>
         </var17>
         <var18>
            <xslt:value-of select="$var18 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var18 instance of xs:string"/>
         </var18>
         <var19>
            <xslt:value-of select="$var19 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var19 instance of xs:time"/>
         </var19>
         <var20>
            <xslt:value-of select="$var20 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var20 instance of xs:normalizedString"/>
         </var20>
         <var21>
            <xslt:value-of select="$var21 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var21 instance of xs:token"/>
         </var21>
         <var22>
            <xslt:value-of select="$var22 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var22 instance of xs:language"/>
         </var22>
         <var23>
            <xslt:value-of select="$var23 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var23 instance of xs:Name"/>
         </var23>
         <var24>
            <xslt:value-of select="$var24 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var24 instance of xs:NCName"/>
         </var24>
         <var25>
            <xslt:value-of select="$var25 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var25 instance of xs:NMTOKEN"/>
         </var25>
         <var26>
            <xslt:value-of select="$var26 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var26 instance of xs:int"/>
         </var26>
         <var27>
            <xslt:value-of select="$var27 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var27 instance of xs:nonPositiveInteger"/>
         </var27>
         <var28>
            <xslt:value-of select="$var28 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var28 instance of xs:positiveInteger"/>
         </var28>
         <var29>
            <xslt:value-of select="$var29 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var29 instance of xs:nonNegativeInteger"/>
         </var29>
         <var30>
            <xslt:value-of select="$var30 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var30 instance of xs:negativeInteger"/>
         </var30>
         <var31>
            <xslt:value-of select="$var31 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var31 instance of xs:long"/>
         </var31>
         <var32>
            <xslt:value-of select="$var32 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var32 instance of xs:unsignedLong"/>
         </var32>
         <var33>
            <xslt:value-of select="$var33 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var33 instance of xs:unsignedInt"/>
         </var33>
         <var34>
            <xslt:value-of select="$var34 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var34 instance of xs:short"/>
         </var34>
         <var35>
            <xslt:value-of select="$var35 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var35 instance of xs:unsignedShort"/>
         </var35>
         <var36>
            <xslt:value-of select="$var36 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var36 instance of xs:byte"/>
         </var36>
         <var37>
            <xslt:value-of select="$var37 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var37 instance of xs:unsignedByte"/>
         </var37>
         <var38>
            <xslt:value-of select="$var38 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var38 instance of xs:ID"/>
         </var38>
         <var39>
            <xslt:value-of select="$var39 instance of xs:anyAtomicType"/>
            <xslt:value-of select="$var39 instance of xs:IDREF"/>
         </var39>
      </out>
   </xslt:template>
</xslt:transform>
