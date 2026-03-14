<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <t:output method="xml"/>
   <!-- Purpose: Test to verify that if @mode is specified with value '#current' 
  				in <xsl:apply-templates/>, the value is carried automatically through 
				built-in rules to explicit templates for descendants at any level in the tree.-->

   <t:template match="/">
      <out>
         <t:apply-templates mode="#current"/>
      </out>
   </t:template>

   <t:template match="empty">
empty-CORRECT!
</t:template>

   <t:template match="sub-empty">
sub-empty-CORRECT!
</t:template>

   <t:template match="comment()" mode="newmode">
comment-CORRECT!
</t:template>

   <t:template match="processing-instruction()" mode="othermode">
processing-instruction-CORRECT!
</t:template>

   <t:template match="text()" mode="good">
text-CORRECT!
</t:template>

   <t:template match="@*">
This should not Print
</t:template>
</t:transform>
