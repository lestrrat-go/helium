<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test the "required" attribute on a template parameter where the "select" attribute is not empty. -->

   <t:template match="doc">
      <out>
         <t:call-template name="boo">
            <t:with-param name="par1" select="'defaultValue'"/>
         </t:call-template>
      </out>
   </t:template>

   <t:template name="boo">
      <t:param name="par1" select="'oops'" required="yes"/> par1= <t:value-of select="$par1"/>
   </t:template>
</t:transform>
