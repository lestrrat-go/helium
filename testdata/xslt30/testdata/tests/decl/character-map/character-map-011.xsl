<?xml version="1.0" encoding="ISO-8859-1"?>
<stylesheet version="2.0" xmlns="http://www.w3.org/1999/XSL/Transform">

<?spec xslt#character-maps?>

    <!-- test character maps with a default namespace -->

   	<output encoding="ISO-8859-1" method="xml" use-character-maps="s"/>  
   	
	<character-map name="s">
		<output-character character="a" string="b"/>
	</character-map> 	
 
	<template match="/">
		<x xmlns="">a</x>
	</template>

</stylesheet>