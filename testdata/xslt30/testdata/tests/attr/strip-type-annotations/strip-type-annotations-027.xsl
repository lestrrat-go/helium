<?xml version="1.0" encoding="UTF-8"?>
<xsl:package xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:my="http://ns.example.com/strip-type"
   xmlns:xsl="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs my"
   input-type-annotations="strip" version="2.0">
   <!-- Purpose: Test that type annotations from the initial source tree are retained if an included stylesheet has 
  				@input-type-annotations="unspecified". The primary stylesheet has @input-type-annotations="strip".
        Variant of -010 but using xsl:package rather than xsl:transform.
   -->

   <xsl:include href="import_unspecified.xsl"/>

   <xsl:import-schema namespace="http://ns.example.com/strip-type"
      schema-location="varietyOfTypes.xsd"/>
   
   <xsl:mode/>

   <xsl:template match="/my:doc">
      <out>
         <primary>
            <E1>
               <xsl:value-of select="elem-anyURI instance of element(*, xs:anyURI)"/>
               <xsl:value-of select="elem-anyURI instance of element(*, xs:untyped)"/>
            </E1>
            <E2>
               <xsl:value-of select="elem-boolean instance of element(*, xs:boolean)"/>
               <xsl:value-of select="elem-boolean instance of element(*, xs:untyped)"/>
            </E2>
            <E3>
               <xsl:value-of select="elem-date instance of element(*, xs:date)"/>
               <xsl:value-of select="elem-date instance of element(*, xs:untyped)"/>
            </E3>
            <E4>
               <xsl:value-of select="elem-double instance of element(*, xs:double)"/>
               <xsl:value-of select="elem-double instance of element(*, xs:untyped)"/>
            </E4>
            <E5>
               <xsl:value-of select="elem-duration instance of element(*, xs:duration)"/>
               <xsl:value-of select="elem-duration instance of element(*, xs:untyped)"/>
            </E5>
            <E6>
               <xsl:value-of select="elem-QName instance of element(*, xs:QName)"/>
               <xsl:value-of select="elem-QName instance of element(*, xs:untyped)"/>
            </E6>
            <E7>
               <xsl:value-of select="elem-string instance of element(*, xs:string)"/>
               <xsl:value-of select="elem-string instance of element(*, xs:untyped)"/>
            </E7>
            <E8>
               <xsl:value-of select="elem-NCName instance of element(*, xs:NCName)"/>
               <xsl:value-of select="elem-NCName instance of element(*, xs:untyped)"/>
            </E8>
            <E9>
               <xsl:value-of select="elem-NMTOKEN instance of element(*, xs:NMTOKEN)"/>
               <xsl:value-of select="elem-NMTOKEN instance of element(*, xs:untyped)"/>
            </E9>
            <E10>
               <xsl:value-of select="elem-int instance of element(*, xs:int)"/>
               <xsl:value-of select="elem-int instance of element(*, xs:untyped)"/>
            </E10>
            <E11>
               <xsl:value-of
                  select="elem-nonPositiveInteger instance of element(*, xs:nonPositiveInteger)"/>
               <xsl:value-of select="elem-nonPositiveInteger instance of element(*, xs:untyped)"/>
            </E11>
            <E12>
               <xsl:value-of select="elem-long instance of element(*, xs:long)"/>
               <xsl:value-of select="elem-long instance of element(*, xs:untyped)"/>
            </E12>
            <E13>
               <xsl:value-of select="elem-unsignedLong instance of element(*, xs:unsignedLong)"/>
               <xsl:value-of select="elem-unsignedLong instance of element(*, xs:untyped)"/>
            </E13>
            <E14>
               <xsl:value-of select="simple-derived instance of element(*, my:partNumberType)"/>
               <xsl:value-of select="simple-derived instance of element(*, xs:untyped)"/>
            </E14>
            <E15>
               <xsl:value-of select="complex-derived instance of element(*, my:personType)"/>
               <xsl:value-of select="complex-derived instance of element(*, xs:untyped)"/>
            </E15>
            <E16>
               <xsl:value-of select="complex-derived/firstName instance of element(*, xs:string)"/>
               <xsl:value-of select="complex-derived/firstName instance of element(*, xs:untyped)"/>
            </E16>
            <E17>
               <xsl:value-of select="complex-derived/familyName instance of element(*, xs:string)"/>
               <xsl:value-of select="complex-derived/familyName instance of element(*, xs:untyped)"
               />
            </E17>
            <E19>
               <xsl:value-of select="complex-derived/@age instance of attribute(*, xs:integer)"/>
               <xsl:value-of
                  select="complex-derived/@age instance of attribute(*, xs:untypedAtomic)"/>
            </E19>
            <E20>
               <xsl:value-of select="list-builtin instance of element(*, xs:NMTOKENS)"/>
               <xsl:value-of select="list-builtin instance of element(*, xs:untyped)"/>
            </E20>
            <E21>
               <xsl:value-of select="list-derived instance of element(*, my:myListType)"/>
               <xsl:value-of select="list-derived instance of element(*, xs:untyped)"/>
            </E21>
            <E22>
               <xsl:value-of select="union-derived[1] instance of element(*, my:partIntegerUnion)"/>
               <xsl:value-of select="union-derived[1] instance of element(*, xs:untyped)"/>
            </E22>
            <E23>
               <xsl:value-of select="union-derived[2] instance of element(*, my:partIntegerUnion)"/>
               <xsl:value-of select="union-derived[2] instance of element(*, xs:untyped)"/>
            </E23>
         </primary>
         <xsl:call-template name="temp"/>
      </out>
   </xsl:template>
</xsl:package>
