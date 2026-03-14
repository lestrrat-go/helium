<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:my="http://ns.example.com/strip-type"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs my"
                input-type-annotations="strip"
                version="2.0">
<!-- Purpose: Show that when @input-type-annotations="strip" the type of all attribute nodes is changed 
  				to xs:untypedAtomic. User-defined simple schema types are used.-->

   <xslt:import-schema namespace="http://ns.example.com/strip-type"
                       schema-location="varietyOfTypes.xsd"/>

   <xslt:template match="/my:userNode">
      
      <out>
         
         <E1>
            <xslt:value-of select="data(@specialPart) instance of my:partNumberType"/>
            <xslt:value-of select="@specialPart instance of attribute(*, my:partNumberType)"/>
            <xslt:value-of select="data(@specialPart) instance of xs:untypedAtomic"/>
            <xslt:value-of select="@specialPart instance of attribute(*, xs:untypedAtomic)"/>
         </E1>
         
         <E2>
            <xslt:value-of select="@listParts instance of attribute(*, my:myListType)"/>
            <xslt:value-of select="data(@listParts) instance of xs:untypedAtomic"/>
            <xslt:value-of select="@listParts instance of attribute(*, xs:untypedAtomic)"/>
         </E2>
         
         <E3>
            <xslt:value-of select="data(complexMixedUserElem/@age) instance of xs:integer"/>
            <xslt:value-of select="complexMixedUserElem/@age instance of attribute(*, xs:integer)"/>
            <xslt:value-of select="data(complexMixedUserElem/@age) instance of xs:untypedAtomic"/>
            <xslt:value-of select="complexMixedUserElem/@age instance of attribute(*, xs:untypedAtomic)"/>
         </E3>
         
         <E4>
            <xslt:value-of select="onlyAttributes/@colors instance of attribute(*, my:myListType)"/>
            <xslt:value-of select="data(onlyAttributes/@colors) instance of xs:untypedAtomic"/>
            <xslt:value-of select="onlyAttributes/@colors instance of attribute(*, xs:untypedAtomic)"/>
         </E4>
         
         <E5>
            <xslt:value-of select="data(onlyAttributes/@specialPart) instance of my:partNumberType"/>
            <xslt:value-of select="onlyAttributes/@specialPart instance of attribute(*, my:partNumberType)"/>
            <xslt:value-of select="data(onlyAttributes/@specialPart) instance of xs:untypedAtomic"/>
            <xslt:value-of select="onlyAttributes/@specialPart instance of attribute(*, xs:untypedAtomic)"/>
         </E5>
         
      </out>
   </xslt:template>
</xslt:transform>
