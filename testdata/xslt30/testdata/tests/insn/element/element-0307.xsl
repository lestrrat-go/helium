<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses different combinations of xsl:namespace and xmlns 
  				attribute on an xsl:element instruction with prefixed and unprefixed QNames 
  				to test the effect of the namespace on the newly created element.-->

   <t:template match="doc">
		    <out>
         <t:text>
</t:text>
         <t:element xmlns:ns="http://www.examples.com/e1" name="ns:e1">
   			   	  <t:attribute name="a1">
   					       <t:value-of select="1"/>
   				     </t:attribute>
   			   </t:element>
         <t:text>
</t:text>
         <t:element xmlns:ns="http://www.examples.com/actualNS"
                    name="ns:e2"
                    namespace="http://www.examples.com/fakeNS"/>
         <t:text>
</t:text>
         <t:element name="ns:e3" namespace="http://www.examples.com/e3">
   				     <t:call-template name="makeAttr"/>
	   		   </t:element>
         <t:text>
</t:text>
         <t:element xmlns="http://www.examples.com/e4" name="e4"/>
         <t:text>
</t:text>
         <t:element xmlns="http://www.examples.com/actualNS"
                    name="e5"
                    namespace="http://www.examples.com/fakeNS"/>
         <t:text>
</t:text>
         <t:element name="e6" namespace="http://www.examples.com/e6"/>
      </out>
	  </t:template>

   <t:template name="makeAttr">
		    <t:attribute name="a2" select="'2'"/>
	  </t:template>
</t:transform>
