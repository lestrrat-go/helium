<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that fn:doc-available always returns false within 
  				use-when, even when it would return true elsewhere in the stylesheet. -->

   <t:variable name="v" as="element() *">
      <a>
         <b/>
         <c>
            <d/>
         </c>
      </a>	
   </t:variable>

   <t:template match="/">
      <out>
         <t:call-template name="temp"/>
         <t:value-of select="doc-available($v)" use-when="doc-available('')"/>
      </out>
   </t:template>

   <t:template name="temp">
	     <t:value-of select="doc-available('')"/>
   </t:template>
</t:transform>
