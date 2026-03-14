<?xml version="1.0" encoding="UTF-8"?>
<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:xslt="http://www.w3.org/1999/XSL/Transform" exclude-result-prefixes="xs"
   input-type-annotations="strip" version="2.0">
   <!-- Purpose: Show that when @input-type-annotations="strip", the typed value of a node is changed to 
  				be equal to its string value and the type annotation of the value is changed to xs:untypedAtomic.-->

   <xslt:template match="/doc-striptype">
      <out>
         <xslt:value-of select="data(elem-integer),
            string(elem-integer),
            data(elem-integer)=string(elem-integer),
            data(elem-integer) instance of xs:integer,
            data(elem-integer) instance of xs:untypedAtomic" separator="|"/>
      </out>
   </xslt:template>
</xslt:transform>
