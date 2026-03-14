<xslt:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
                xmlns:xslt="http://www.w3.org/1999/XSL/Transform"
                exclude-result-prefixes="xs"
                version="2.0">

<!-- Test with NOTATION declared in a DTD and an attribute pointing to it. 
  				Show that the type of the attribute is xs:untypedAtomic and not xs:NOTATION.  -->

   <xslt:template match="/doc">
      <out>
         <xslt:value-of select="@bar instance of attribute(*, xs:NOTATION)"/>
      </out>
   </xslt:template>
</xslt:transform>
