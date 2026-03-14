<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://ns.example.com/strip-type"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                input-type-annotations="unspecified"
                version="2.0">
<!-- Purpose: Test that type annotations are stripped from the initial source tree if imported stylesheet has 
  				@input-type-annotations="strip". The primary stylesheet has @input-type-annotations= "unspecified".-->

   <xslt:import href="import_strip.xsl"/>

   <xslt:import-schema namespace="http://ns.example.com/strip-type"
                       schema-location="varietyOfTypes.xsd"/>

   <xslt:template match="/my:doc">
	     
      <out>
         
         <primary>
            
            <E1>
               <xslt:value-of select="elem-anyURI instance of element(*, xs:anyURI)"/>
               <xslt:value-of select="elem-anyURI instance of element(*, xs:untyped)"/>
            </E1>
            
            <E2>
               <xslt:value-of select="elem-boolean instance of element(*, xs:boolean)"/>
               <xslt:value-of select="elem-boolean instance of element(*, xs:untyped)"/>
            </E2>
            
            <E3>
               <xslt:value-of select="elem-date instance of element(*, xs:date)"/>
               <xslt:value-of select="elem-date instance of element(*, xs:untyped)"/>
            </E3>
            
            <E4>
               <xslt:value-of select="elem-double instance of element(*, xs:double)"/>
               <xslt:value-of select="elem-double instance of element(*, xs:untyped)"/>
            </E4>
            
            <E5>
               <xslt:value-of select="elem-duration instance of element(*, xs:duration)"/>
               <xslt:value-of select="elem-duration instance of element(*, xs:untyped)"/>
            </E5>
            
            <E6>
               <xslt:value-of select="elem-QName instance of element(*, xs:QName)"/>
               <xslt:value-of select="elem-QName instance of element(*, xs:untyped)"/>
            </E6>
            
            <E7>
               <xslt:value-of select="elem-string instance of element(*, xs:string)"/>
               <xslt:value-of select="elem-string instance of element(*, xs:untyped)"/>
            </E7>
            
            <E8>
               <xslt:value-of select="elem-NCName instance of element(*, xs:NCName)"/>
               <xslt:value-of select="elem-NCName instance of element(*, xs:untyped)"/>
            </E8>
            
            <E9>
               <xslt:value-of select="elem-NMTOKEN instance of element(*, xs:NMTOKEN)"/>
               <xslt:value-of select="elem-NMTOKEN instance of element(*, xs:untyped)"/>
            </E9>
            
            <E10>
               <xslt:value-of select="elem-int instance of element(*, xs:int)"/>
               <xslt:value-of select="elem-int instance of element(*, xs:untyped)"/>
            </E10>
            
            <E11>
               <xslt:value-of select="elem-nonPositiveInteger instance of element(*, xs:nonPositiveInteger)"/>
               <xslt:value-of select="elem-nonPositiveInteger instance of element(*, xs:untyped)"/>
            </E11>
            
            <E12>
               <xslt:value-of select="elem-long instance of element(*, xs:long)"/>
               <xslt:value-of select="elem-long instance of element(*, xs:untyped)"/>
            </E12>
            
            <E13>
               <xslt:value-of select="elem-unsignedLong instance of element(*, xs:unsignedLong)"/>
               <xslt:value-of select="elem-unsignedLong instance of element(*, xs:untyped)"/>
            </E13>
            
            <E14>
               <xslt:value-of select="simple-derived instance of element(*, my:partNumberType)"/>
               <xslt:value-of select="simple-derived instance of element(*, xs:untyped)"/>
            </E14>
            
            <E15>
               <xslt:value-of select="complex-derived instance of element(*, my:personType)"/>
               <xslt:value-of select="complex-derived instance of element(*, xs:untyped)"/>
            </E15>
            
            <E16>
               <xslt:value-of select="complex-derived/firstName instance of element(*, xs:string)"/>
               <xslt:value-of select="complex-derived/firstName instance of element(*, xs:untyped)"/>
            </E16>
            
            <E17>
               <xslt:value-of select="complex-derived/familyName instance of element(*, xs:string)"/>
               <xslt:value-of select="complex-derived/familyName instance of element(*, xs:untyped)"/>
            </E17>
            
            <E19>
               <xslt:value-of select="complex-derived/@age instance of attribute(*, xs:integer)"/>
               <xslt:value-of select="complex-derived/@age instance of attribute(*, xs:untypedAtomic)"/>
            </E19>
            
            <E20>
               <xslt:value-of select="list-builtin instance of element(*, xs:NMTOKENS)"/>
               <xslt:value-of select="list-builtin instance of element(*, xs:untyped)"/>
            </E20>
            
            <E21>
               <xslt:value-of select="list-derived instance of element(*, my:myListType)"/>
               <xslt:value-of select="list-derived instance of element(*, xs:untyped)"/>
            </E21>
            
            <E22>
               <xslt:value-of select="union-derived[1] instance of element(*, my:partIntegerUnion)"/>
               <xslt:value-of select="union-derived[1] instance of element(*, xs:untyped)"/>
            </E22>
            
            <E23>
               <xslt:value-of select="union-derived[2] instance of element(*, my:partIntegerUnion)"/>
               <xslt:value-of select="union-derived[2] instance of element(*, xs:untyped)"/>
            </E23>
            
         </primary>
         <xslt:call-template name="temp"/>
      </out>
   </xslt:template>
</xslt:transform>
