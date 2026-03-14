<?xml version="1.0"?>
<!DOCTYPE t:stylesheet [ 
<!ENTITY br "<t:text>&#xa;</t:text>">
<!ENTITY % ext SYSTEM "entity.ent">%ext;
]>

<t:stylesheet xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test referencing an internal whitespace entity 
  				and an external parameter whitespace entity defined 
  				in a DTD. -->

  <t:template match="/doc">
    <out>
      <a><t:text>&#xa;</t:text> &br;</a>
      <b>&space;test</b>
    </out>
  </t:template>

</t:stylesheet>
