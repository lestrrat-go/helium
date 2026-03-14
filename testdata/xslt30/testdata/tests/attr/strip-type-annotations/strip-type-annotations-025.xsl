<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                input-type-annotations="strip"
                version="2.0">
<!-- Purpose: Test with xsl:stylesheet and @input-type-annotations="strip". Input file is not validated.-->

   <xslt:template match="/doc">		    
      <out>        
         <E1>
            <xslt:value-of select="data(elem-anyURI) instance of xs:anyURI"/>
            <xslt:value-of select="data(elem-anyURI) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-anyURI instance of element(*, xs:untyped)"/>
         </E1>
         
         <E2>
            <xslt:value-of select="data(elem-boolean) instance of xs:boolean"/>
            <xslt:value-of select="data(elem-boolean) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-boolean instance of element(*, xs:untyped)"/>
         </E2>
         
         <E3>
            <xslt:value-of select="data(elem-date) instance of xs:date"/>
            <xslt:value-of select="data(elem-date) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-date instance of element(*, xs:untyped)"/>
         </E3>
         
         <E4>
            <xslt:value-of select="data(elem-double) instance of xs:double"/>
            <xslt:value-of select="data(elem-double) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-double instance of element(*, xs:untyped)"/>
         </E4>
         
         <E5>
            <xslt:value-of select="data(elem-duration) instance of xs:duration"/>
            <xslt:value-of select="data(elem-duration) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-duration instance of element(*, xs:untyped)"/>
         </E5>
         
         <E6>
            <xslt:value-of select="data(elem-QName) instance of xs:QName"/>
            <xslt:value-of select="data(elem-QName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-QName instance of element(*, xs:untyped)"/>
         </E6>
         
         <E7>
            <xslt:value-of select="data(elem-string) instance of xs:string"/>
            <xslt:value-of select="data(elem-string) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-string instance of element(*, xs:untyped)"/>
         </E7>
         
         <E8>
            <xslt:value-of select="data(elem-NCName) instance of xs:string"/>
            <xslt:value-of select="data(elem-NCName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-NCName instance of element(*, xs:untyped)"/>
         </E8>
         
         <E9>
            <xslt:value-of select="data(elem-NMTOKEN) instance of xs:string"/>
            <xslt:value-of select="data(elem-NMTOKEN) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-NMTOKEN instance of element(*, xs:untyped)"/>
         </E9>
         
         <E10>
            <xslt:value-of select="data(elem-int) instance of xs:integer"/>
            <xslt:value-of select="data(elem-anyURI) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-anyURI instance of element(*, xs:untyped)"/>
         </E10>
         
         <E11>
            <xslt:value-of select="data(elem-nonPositiveInteger) instance of xs:integer"/>
            <xslt:value-of select="data(elem-nonPositiveInteger) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-nonPositiveInteger instance of element(*, xs:untyped)"/>
         </E11>
         
         <E12>
            <xslt:value-of select="data(elem-long) instance of xs:integer"/>
            <xslt:value-of select="data(elem-long) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-long instance of element(*, xs:untyped)"/>
         </E12>
         
         <E13>
            <xslt:value-of select="data(elem-unsignedLong) instance of xs:integer"/>
            <xslt:value-of select="data(elem-unsignedLong) instance of xs:untypedAtomic"/>
            <xslt:value-of select="elem-unsignedLong instance of element(*, xs:untyped)"/>
         </E13>
         
         <E14>
            <xslt:value-of select="data(complex-derived/firstName) instance of xs:string"/>
            <xslt:value-of select="data(complex-derived/firstName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="complex-derived/firstName instance of element(*, xs:untyped)"/>
         </E14>
         
         <E15>
            <xslt:value-of select="data(complex-derived/familyName) instance of xs:string"/>
            <xslt:value-of select="data(complex-derived/familyName) instance of xs:untypedAtomic"/>
            <xslt:value-of select="complex-derived/familyName instance of element(*, xs:untyped)"/>
         </E15>
         
         <E16>
            <xslt:value-of select="complex-derived/@age instance of xs:integer"/>
            <xslt:value-of select="data(complex-derived/@age) instance of xs:untypedAtomic"/>
            <xslt:value-of select="complex-derived/@age instance of attribute(*, xs:untypedAtomic)"/>
         </E16>
         
         <E17>
            <xslt:value-of select="list-builtin instance of element(*, xs:string)"/>
            <xslt:value-of select="data(list-builtin) instance of xs:untypedAtomic"/>
            <xslt:value-of select="list-builtin instance of element(*, xs:untyped)"/>
         </E17>
         
      </out>
   </xslt:template>
</xslt:transform>
