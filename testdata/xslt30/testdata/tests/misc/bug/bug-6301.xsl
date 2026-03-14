<?xml version="1.0" encoding="UTF-8"?>
<stylesheet version="2.0" xmlns="http://www.w3.org/1999/XSL/Transform"
xmlns:xs="http://www.w3.org/2001/XMLSchema">

<?spec xpath#id-for-expressions?>
<!-- Bug 788748 reported by Oliver Becker, ClassCastException -->

<template match="/">
  <element name="out" namespace="">
    <value-of select="2,
for $x in (3 to 1000) 
return 
  if (count(for $y in (2 to xs:integer($x div 2))
            return 
              if (floor($x div $y)*$y=$x)
              then $y 
              else ()
      )=0) 
  then $x 
  else ()" 
separator=", " />
  </element>
</template>

</stylesheet>
