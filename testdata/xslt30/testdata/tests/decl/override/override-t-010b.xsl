<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:xs="http://www.w3.org/2001/XMLSchema"
   xmlns:t="http://www.w3.org/1999/XSL/Transform" 
   exclude-result-prefixes="xs" version="3.0">
   <!-- Purpose:  It should not be possible to override a template with no declared visibility (default is private)  -->




   <t:use-package name="http://www.w3.org/xslts/override-t-010a" package-version="3.0">

      <t:override>
         <t:template name="a">
            <a2>
               <t:value-of select="current-date()"/>
            </a2>
            <a1>
               <t:call-template name="t:original"/>
            </a1>
         </t:template>
      </t:override>

   </t:use-package>
   
   <t:template name="t:initial-template" visibility="public">
      <t:call-template name="a"/>
   </t:template>

</t:transform>
