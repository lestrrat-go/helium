<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0" xmlns:f="http://f.com/"
   xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="#all" expand-text="true">
   <!-- Purpose: Test with UCA collation: effect of the "alternate" option (which controls handling of space and punctuation).-->
   
   <!-- 
      With non-ignorable, spaces and hyphens should be treated as ordinary characters with as much significance as any other character.
      
      With blanked, spaces and hyphens should be ignored completely.
      
      With shifted:
       * Sorts all variable characters [=hyphen, space, etc] less-than (before) regular characters.
       * Appending a variable character makes a string sort greater-than the string without it.
       * Inserting a variable character makes a string sort less-than the string without it.
       * Inserting a variable character earlier in a string makes it sort less-than inserting the variable character later in the string.
   -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="in" as="xs:string*"
      select="tokenize('deluge Deluge delug delu-ge de-luge deluge-')"/>

   <t:template name="t:initial-template">
      <out>
         <primary>
            <non-ignorable>{f:sort("non-ignorable","primary")}</non-ignorable>
            <shifted>{f:sort("shifted","primary")}</shifted>
            <blanked>{f:sort("blanked","primary")}</blanked>
         </primary>
         <secondary>
            <non-ignorable>{f:sort("non-ignorable","secondary")}</non-ignorable>
            <shifted>{f:sort("shifted","secondary")}</shifted>
            <blanked>{f:sort("blanked","secondary")}</blanked>
         </secondary>
         <tertiary>
            <non-ignorable>{f:sort("non-ignorable","tertiary")}</non-ignorable>
            <shifted>{f:sort("shifted","tertiary")}</shifted>
            <blanked>{f:sort("blanked","tertiary")}</blanked>
         </tertiary>
      </out>
   </t:template>

   <t:function name="f:sort">
      <t:param name="option" as="xs:string"/>
      <t:param name="strength" as="xs:string"/>
      <t:perform-sort select="$in">
         <t:sort select="."
            collation="http://www.w3.org/2013/collation/UCA?lang=en;strength={$strength};alternate={$option}"/>
      </t:perform-sort>
   </t:function>
   
   <t:template match="/" mode="bbb">
      <t:value-of select="@x"/>
   </t:template>
</t:transform>
